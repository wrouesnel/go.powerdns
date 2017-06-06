#!/bin/sh
# Symlink the .git hooks to the support directory

ln -sf ../../tools/pre-commit ./.git/hooks/pre-commit
