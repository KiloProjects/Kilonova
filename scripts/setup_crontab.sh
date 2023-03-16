#!/usr/bin/bash


DESTINATION="/home/alexv/data_cleanup.py"
CLEANUP_PATH="/data/kilonova/subtests"

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
cp "$SCRIPT_DIR/data_cleanup.py" "$DESTINATION"

new_entry="*/30 * * * * /usr/bin/sudo /usr/bin/python3 $DESTINATION $CLEANUP_PATH >> /home/alexv/data_cleanup.log 2>&1"

(crontab -l ; echo "$new_entry") | crontab -

# Check the result
if [ $? -eq 0 ]; then
  echo "New crontab entry added successfully."
else
  echo "Failed to add new crontab entry."
fi