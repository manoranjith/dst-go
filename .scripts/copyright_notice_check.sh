#!/bin/bash

bold=`tput bold`
bold_highlight=`tput setab 120 bold`
reset=`tput sgr0`

template="$(dirname $0)/copyright_notice_template.txt"
n=$(wc -l $template | awk '{ print $1 }')

function check_copyright_notice() {
  start_line=1
  end_line=$n
  f=$1
  diff_output=$(diff --color=always <(sed -ne "${start_line},${end_line}p" $f | \
  sed "s/20\(19\|2[0-9]\)/20XX/") $template)
  [ $? -ne 0 ] && echo -e "${bold}\nIn file $f\n$diff_output"
}

exit_status=0
for f in $(find . -name "*.go"); do
  # Skip generated files, Identified by DO NOT EDIT phrase in line 1.
  if ! sed -ne '1,1p' $f | grep "DO NOT EDIT." -q; then
    check_copyright_notice $f
  fi
  [ $? -ne 0 ] && exit_status=1
done
[ $exit_status -ne 0 ] && echo -e "$bold_highlight\n\nHints to fix:$reset\n
1. The actual text in the file is marked red and the expected content
   is marked green.
3. Number before the character a/c/d (in the text above each change)
   is the line number in the file.\n"
exit $exit_status
