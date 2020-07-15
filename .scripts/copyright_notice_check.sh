#!/bin/bash

template="$(dirname $(readlink -f $0))/copyright_notice_template.txt"
n=$(wc -l $template | cut -d ' ' -f 1)

function check_copyright_notice() {
  start_line=1
  end_line=$n
  f=$1
  diff $template <(sed -ne "${start_line},${end_line}p" $f | \
  sed "s/20\(19\|2[0-9]\)/20XX/")
  [ $? -ne 0 ] && echo -e "$f\n"
}

# Its trickier to pass custom bash functions as argument to -exec 
# option of find, so pass the list of files, recursively as args
# to this script.
#
# During the next invocation, since count of arguments is not zero,
# copyright notice check is executed for each file in the list.
if [ $# -eq 0 ]; then
  find . -path "./internal/mocks/*" -prune -o -name "*.go" -exec $0 {} +
else
  code=0
  for f in "$@"; do
    check_copyright_notice $f
    [ $? -ne 0 ] && code=1
  done
  exit $code
fi
