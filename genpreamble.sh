#!/bin/sh

ffmpeg \
  -use_wallclock_as_timestamps 1 \
  -t 1 -r 50 -f lavfi -i color=c=black:s=1280x720 \
  -t 1 -f lavfi -i anullsrc=r=48000:cl=stereo \
  -pix_fmt:v yuv420p -color_range:v tv -color_primaries:v bt709 -colorspace bt709 -color_trc bt709 -profile:v high -b:v 3500000 -c:v libx264 \
  -b:a 128000 -c:a aac \
  -f mpegts -y out.ts
