#!/bin/bash
# Copyright Contributors to the Open Cluster Management project

# TESTED ON MAC!

# NOTE: When running against a node repo, delete the node_modules directories first!  Then npm ci once all the
#       copyright changes are incorporated.

# set -x
TMP_FILE="tmp_file"

ALL_FILES=$(git ls-files | \
 grep -v -f <(sed 's/\([.|]\)/\\\1/g; s/\?/./g ; s/\*/.*/g' .copyrightignore))

COMMUNITY_COPY_HEADER_FILE="$PWD/build/copyright-header.txt"

if [ ! -f $COMMUNITY_COPY_HEADER_FILE ]; then
  echo "File $COMMUNITY_COPY_HEADER_FILE not found!"
  exit 1
fi

COMMUNITY_COPY_HEADER_STRING=$(cat $COMMUNITY_COPY_HEADER_FILE)

echo "Desired copyright header is: $COMMUNITY_COPY_HEADER_STRING"

# NOTE: Only use one newline or javascript and typescript linter/prettier will complain about the extra blank lines
NEWLINE="\n"

ERROR=false

for FILE in $ALL_FILES
do
    if [[ -d $FILE ]] ; then
        continue
    fi

    COMMENT_START="# "
    COMMENT_END=""

    if [[ $FILE  == *".go" ]]; then
        COMMENT_START="// "
    fi

    if [[ $FILE  == *".ts" || $FILE  == *".tsx" || $FILE  == *".js" ]]; then
        COMMENT_START="/* "
        COMMENT_END=" */"
    fi

    if [[ $FILE  == *".md" ]]; then
        COMMENT_START="\[comment\]: # ( "
        COMMENT_END=" )"
    fi

    if [[ $FILE  == *".html" ]]; then
        COMMENT_START="<!-- "
        COMMENT_END=" -->"
    fi

    if [[ $FILE  == *".go"       \
            || $FILE == *".yaml" \
            || $FILE == *".yml"  \
            || $FILE == *".sh"   \
            || $FILE == *".js"   \
            || $FILE == *".ts"   \
            || $FILE == *".tsx"   \
            || $FILE == *"Dockerfile" \
            || $FILE == *"Makefile"  \
            || $FILE == *"Dockerfile.prow" \
            || $FILE == *"Makefile.prow"  \
            || $FILE == *".gitignore"  \
            || $FILE == *".md"  ]]; then

        COMMUNITY_HEADER_AS_COMMENT="$COMMENT_START$COMMUNITY_COPY_HEADER_STRING$COMMENT_END"

        if ! grep -q "$COMMUNITY_HEADER_AS_COMMENT" "$FILE"; then
            echo "FILE: $FILE:"
            echo -e "\t- Need add Community copyright header to file"
            ERROR=true
        fi
    fi
done

if $ERROR == true 
then
  exit 1
fi
rm -f $TMP_FILE