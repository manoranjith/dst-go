#!/bin/bash

template="$(dirname $0)/copyright_notice_template.txt"
n=$(wc -l $template | awk '{ print $1 }')

function check_copyright_notice() {
  start_line=1
  end_line=$n
  f=$1
  diff $template <(sed -ne "${start_line},${end_line}p" $f | \
  sed "s/20\(19\|2[0-9]\)/20XX/")
  [ $? -ne 0 ] && echo -e "$f\n"
}

exit_status=0
for f in $(find . -name "*.go"); do
  # Skip generated files, Identified by DO NOT EDIT phrase in line 1.
  if ! sed -ne '1,1p' $f | grep "DO NOT EDIT." -q; then
    check_copyright_notice $f
  fi
  [ $? -ne 0 ] && exit_status=1
done
exit $exit_status
