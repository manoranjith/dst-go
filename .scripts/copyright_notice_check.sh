#!/bin/bash

template="$(dirname $0)/copyright_notice_template.txt"
n=$(wc -l $template | awk '{ print $1 }')

function check_copyright_notice() {
  start_line=1
  end_line=$n
  f=$1
  diff_output=$(diff --color=always <(sed -ne "${start_line},${end_line}p" $f | \
  sed "s/20\(19\|2[0-9]\)/20XX/") $template)
  [ $? -ne 0 ] && echo -e "\033[1m\n$f\033[0m" && echo "$diff_output"
}

exit_status=0
for f in $(find . -name "*.go"); do
  # Skip generated files, Identified by DO NOT EDIT phrase in line 1.
  if ! sed -ne '1,1p' $f | grep "DO NOT EDIT." -q; then
    check_copyright_notice $f
  fi
  [ $? -ne 0 ] && exit_status=1
done
[ $exit_status -ne 0 ] && echo -e "\e[30;1;47m\n\nTo fix,\
 replace the red lines marked \"\<\" with (green lines marked \"\>\". \033[0m"
exit $exit_status
