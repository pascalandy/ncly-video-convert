#!/usr/bin/env bash
# acceptance.sh [FIXTURE] runs the command of ## Acceptance in spec.md as an
# agent would, with --json and --no-input. just acceptance runs it. It installs
# the extension and the ncly of the Nicely version that go.mod pins into scratch
# folders, so the install of the user stays as it is. With a fixture, it works
# on a copy and checks that the checksum of the fixture does not change.
# Without one, it makes a clip with ffmpeg, turned 180 degrees on screen.
set -Eeuo pipefail

die() {
	echo "acceptance: $*" >&2
	exit 1
}

sum() {
	if command -v sha256sum >/dev/null; then sha256sum "$1"; else shasum -a 256 "$1"; fi | cut -d ' ' -f 1
}

command=(video convert)
fixture="${1:-}"
if [[ -n "$fixture" ]]; then
	[[ -f "$fixture" ]] || die "$fixture is no file"
	[[ "$fixture" == /* ]] || fixture="$PWD/$fixture"
fi
cd "$(dirname "${BASH_SOURCE[0]}")"
scratch="$(mktemp -d "${TMPDIR:-/tmp}/ncly-video-convert-acceptance.XXXXXX")"
trap 'rm -rf "$scratch"' EXIT
work="$scratch/work"
mkdir -p "$scratch/bin" "$work"

# A shim of mise reads its config, state, and installs under the XDG folders,
# which move below. So each program is found first, and its folder goes first
# in PATH.
path="$scratch/bin"
for program in go just HandBrakeCLI ffmpeg ffprobe; do
	found="$(mise which "$program" 2>/dev/null || command -v "$program")" || die "$program is not on PATH"
	path="$path:${found%/*}"
done
export PATH="$path:$PATH"
# Go builds before XDG_CACHE_HOME moves, so it keeps the caches of the user.
go build -o "$scratch/bin/ncly" github.com/pascalandy/nicely/cmd/ncly
XDG_DATA_HOME="$scratch/data" just install
export XDG_DATA_HOME="$scratch/data" XDG_CONFIG_HOME="$scratch/config" XDG_STATE_HOME="$scratch/state" XDG_CACHE_HOME="$scratch/cache"

if [[ -n "$fixture" ]]; then
	before="$(sum "$fixture")"
	cp "$fixture" "$work/"
	input="$work/${fixture##*/}"
else
	# A fixed creation_time keeps the output the same from run to run, since
	# HandBrake writes the clock in its place.
	ffmpeg -nostdin -v error -f lavfi -i testsrc2=size=1280x720:rate=30 -f lavfi -i sine=frequency=440:sample_rate=48000 \
		-t 4 -c:v libx264 -pix_fmt yuv420p -c:a aac -shortest "$scratch/flat.mp4"
	ffmpeg -nostdin -v error -display_rotation 180 -i "$scratch/flat.mp4" -c copy -metadata creation_time=2026-01-01T00:00:00Z "$work/clip.mov"
	input="$work/clip.mov"
fi
status=0
(cd "$work" && ncly "${command[@]}" "$input" --json --no-input) >"$scratch/answer" 2>"$scratch/stderr" || status=$?
cat "$scratch/answer"
cat "$scratch/stderr" >&2
if [[ -n "$fixture" && "$(sum "$fixture")" != "$before" ]]; then
	die "the checksum of $fixture changed"
fi
[[ "$status" == 0 ]] || die "ncly ${command[*]} exited $status"
grep -qF '"ok":true' "$scratch/answer" || die 'the answer holds no "ok":true'
# The answer names the output by its physical path, such as /private/var on macOS.
output="$(sed -n 's/.*"output":"\([^"]*\)".*/\1/p' "$scratch/answer")"
want="${input##*/}"
[[ "${output##*/}" == "${want%.*}.converted.mp4" && -f "$output" ]] || die "the answer names no output beside $input"
grep -qE "\"size_bytes\":$(wc -c <"$output" | tr -d ' ')[,}]" "$scratch/answer" || die "size_bytes differs from wc -c of $output"

probe() {
	ffprobe -v error -select_streams "$1" -show_entries "$2" -of default=noprint_wrappers=1:nokey=1 "$3" | head -n 1
}
duration() {
	ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 "$1"
}
# ssim prints the SSIM of one frame, at 1 s, of two videos as the screen shows them.
ssim() {
	ffmpeg -nostdin -ss 1 -i "$1" -ss 1 -i "$2" -lavfi \
		"[0:v]scale=320:320,format=gray,trim=end_frame=1[a];[1:v]scale=320:320,format=gray,trim=end_frame=1[b];[a][b]ssim" \
		-f null - 2>&1 | sed -n 's/.*All:\([0-9.]*\).*/\1/p'
}
# above A B succeeds when the number A is at least B.
above() {
	awk -v a="$1" -v b="$2" 'BEGIN { exit !(a != "" && a + 0 >= b + 0) }'
}

ffmpeg -nostdin -v error -xerror -i "$output" -f null - || die "ffmpeg cannot decode $output"
[[ "$(probe v:0 stream=codec_name "$output")" == h264 ]] || die "the video of $output is not H.264"
[[ "$(probe a:0 stream=codec_name "$output")" == aac ]] || die "$output has no AAC audio track"
# The preset keeps a frame of up to 1920x1080 as the screen shows it.
size="$(probe v:0 stream=width "$input")x$(probe v:0 stream=height "$input")"
case "$(probe v:0 stream_side_data=rotation "$input")" in 90 | -90 | 270 | -270) size="${size#*x}x${size%x*}" ;; esac
got="$(probe v:0 stream=width "$output")x$(probe v:0 stream=height "$output")"
if [[ "${size%x*}" -le 1920 && "${size#*x}" -le 1080 && "$got" != "$size" ]]; then
	die "the frame of $output is $got, want $size"
fi
in_duration="$(duration "$input")" out_duration="$(duration "$output")"
awk -v a="$in_duration" -v b="$out_duration" 'BEGIN { d = a - b; exit !(d <= 0.1 && d >= -0.1) }' ||
	die "the duration of $output, $out_duration s, is not within 0.1 s of $in_duration s"
orientation="$(ssim "$input" "$output")"
above "$orientation" 0.8 || die "$output does not show the input the same way up: SSIM $orientation, want 0.8 or more"

# A second run checks determinism, as the Determinism criterion of qa.md says.
mv "$output" "$work/first.mp4"
(cd "$work" && ncly "${command[@]}" "$input" --json --no-input) >/dev/null 2>"$scratch/stderr" || die "the second run failed: $(cat "$scratch/stderr")"
if [[ "$(sum "$work/first.mp4")" == "$(sum "$output")" ]]; then
	determinism="the same checksum"
else
	frames() {
		ffprobe -v error -count_packets -select_streams v:0 -show_entries stream=nb_read_packets -of csv=p=0 "$1"
	}
	audio() {
		ffmpeg -nostdin -v error -i "$1" -map 0:a -c copy -f streamhash -hash sha256 -
	}
	[[ "$(frames "$work/first.mp4")" == "$(frames "$output")" ]] || die "two runs give a different frame count"
	[[ "$(audio "$work/first.mp4")" == "$(audio "$output")" ]] || die "two runs give a different audio stream"
	runs="$(ffmpeg -nostdin -i "$work/first.mp4" -i "$output" -lavfi "[0:v][1:v]ssim" -f null - 2>&1 | sed -n 's/.*All:\([0-9.]*\).*/\1/p')"
	above "$runs" 0.999 || die "two runs differ: video SSIM $runs, want 0.999 or more"
	determinism="the same frames and audio, and a video SSIM of $runs"
fi

# The domain refuses a still image, which HandBrakeCLI accepts. Until
# nicely#351 lands, the command accepts it too, and this check expects that.
ffmpeg -nostdin -v error -f lavfi -i testsrc2=size=320x240 -frames:v 1 "$work/still.png"
code=0
(cd "$work" && ncly "${command[@]}" "$work/still.png" --json --no-input) >/dev/null 2>"$scratch/refusal" || code=$?
if [[ "$code" == 2 ]] && grep -qF '"code":"VIDEO_CONVERT_UNREADABLE"' "$scratch/refusal"; then
	die "the still image is refused now: require that refusal here, and close nicely#351"
fi
[[ "$code" == 0 ]] || die "$work/still.png: want exit 0 until nicely#351 lands, got exit $code: $(cat "$scratch/refusal")"

echo "acceptance: ncly ${command[*]} passed on $input: H.264 and AAC, $got, $out_duration s, orientation SSIM $orientation; two runs give $determinism; a still image is still accepted (nicely#351)"
