#!/usr/bin/env bash

set -e

# Given input to pick a quote, output a quote from CS Lewis
case $1 in
  cliff)
    echo "When the whole world is running towards a cliff, he who is running in the opposite direction appears to have lost his mind."
    ;;
  difficulties)
    echo "Life with God is not immunity from difficulties, but peace in difficulties."
    ;;
  believing)
    echo "Once people stop believing in God, the problem is not that they will believe in nothing; rather, the problem is that they will believe anything."
    ;;
  tyranny)
    echo "Of all tyrannies, a tyranny sincerely exercised for the good of its victims may be the most oppressive."
    ;;
  evil)
    echo "The greatest evils in the world will not be carried out by men with guns, but by men in suits sitting behind desks."
    ;;
  *)
    echo "Bad input. Please provide a string argument: cliff, difficulties, believing, tyranny, or evil."
    exit 1
    ;;
esac