#!/usr/bin/env python3
"""Update learning timestamp after successful session"""
from datetime import datetime

timestamp_file = "/home/amin/.hermes/learning_timestamp.txt"
with open(timestamp_file, 'w') as f:
    f.write(datetime.now().isoformat())

print(f"✓ Learning timestamp updated: {datetime.now().isoformat()}")
