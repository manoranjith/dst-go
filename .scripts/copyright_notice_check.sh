#!/bin/bash

template="$(dirname $(readlink -f $0))/copyright_notice_template.txt"
n=$(wc -l $template | cut -d ' ' -f 1)

function check_copyright_notice() {
  # if the file has build tag, skip the first two lines and look for
  # copyright notice from line 3.
  if sed -ne '1,1p' $f | grep "build" -q; then
    start_line=$(expr 1 + 2)
    end_line=$(expr $n + 2)
  else
    start_line=1
    end_line=$n
  fi

  if ! get_diff $1 $start_line $end_line  > /dev/null; then
    echo -e "\n$f"
    get_diff $1 $start_line $end_line
  fi
}

function get_diff() {
  start_line=$2
  end_line=$3
  diff $template <(sed -ne "${start_line},${end_line}p" $1 | \
   sed "s/20\(19\|2[0-9]\)/20XX/")
}


# Find cannot execute bash custome functions over the list of files,
# so recursively call this script the list of files as arguments.
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
