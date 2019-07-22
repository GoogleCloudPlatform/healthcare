#!/bin/bash
#
# This script takes in a text file of Google Cloud Credit Coupons, and outputs a
# LaTeX source file with COUPON-CODE replaced by those in the coupon file.
#
# Usage:
#   export COUPONFILE=/path/to/file
#   ./gen_tex.sh | pdflatex --output-directory ~/

cat header.tex
for l in `cat ${COUPONFILE}`
do
    cat body.tex | sed "s#COUPON-CODE#${l}#"
done
cat footer.tex
