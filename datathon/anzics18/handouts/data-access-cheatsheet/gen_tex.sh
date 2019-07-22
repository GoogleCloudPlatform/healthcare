#!/bin/bash
#
# This script takes in a CSV file of the form "user_id,group_id,password", and
# output a LaTeX source file with R-USER and R-PASSWORD replaced by those in the
# password CSV file.
#
# Usage:
#   export PASSFILE=/path/to/file
#   ./gen_tex.sh | pdflatex --output-directory ~/

cat header.tex
for l in `tail -n 120 ${PASSFILE}`
do
    # Extract user_id and password fields
    user_id=$(echo $l | sed 's/,.*$//')
    password=$(echo $l | sed 's/^[^,]*,[^,]*,//')
    cat body.tex | sed -e "s#R-USER#${user_id}#" -e "s#R-PASSWORD#${password}#"
done
cat footer.tex
