#!/bin/bash

# exit_status would be set to 1 in the check_copyright_notice(),
# when the first error is detected.
# it will remain zero, if no errors are detected.
exit_status=0

# Formating directives for printing text.
bold=`tput bold`
reset=`tput sgr0`

template="$(dirname $0)/copyright_notice_template.txt"
n=$(wc -l $template | awk '{ print $1 }')

function check_copyright_notice() {
  start_line=1
  end_line=$n
  f=$1
  diff_output=$(diff --color=always <(sed -ne "${start_line},${end_line}p" $f | \
  sed "s/20\(19\|2[0-9]\)/20XX/") $template)
  if [ $? -ne 0 ]; then
    exit_status=1
    echo -e "${bold}\nIn file $f\n$diff_output"
  fi
}

for f in $(find . -name "*.go"); do
  # Skip generated files, Identified by "DO NOT EDIT" phrase in line 1.
  if ! sed -ne '1,1p' $f | grep "DO NOT EDIT." -q; then
    check_copyright_notice $f
  fi
done

if [ $exit_status -ne 0 ]; then
  echo -e "$bold\n\nHints to fix:$reset\n
1. The actual text in the file is marked red and the expected text
   is marked green.
2. Number before the character a/c/d (in the text above each change)
   is the line number in the file.\n"
fi

exit $exit_status