#!/usr/bin/env python3
"""
Update timestamp for learning session completion
"""
import os
from datetime import datetime

timestamp_file = os.path.expanduser('~/.hermes/learning_timestamp.txt')
os.makedirs(os.path.dirname(timestamp_file), exist_ok=True)

with open(timestamp_file, 'w') as f:
    f.write(datetime.now().isoformat())

print(f"✓ Timestamp updated: {datetime.now().isoformat()}")
