import json, sys, requests

def get_roster(season):
    url = f"https://api-web.nhle.com/v1/roster/UTA/{season}"
    r = requests.get(url)
    return r.json()

season_24 = get_roster("20242025")
season_25 = get_roster("20252026")

print("\n=== UTAH ROSTER COMPARISON ===\n")

# Forwards
print("FORWARDS:")
fwd_24 = {f"{p['firstName']['default']} {p['lastName']['default']}" for p in season_24.get('forwards', [])}
fwd_25 = {f"{p['firstName']['default']} {p['lastName']['default']}" for p in season_25.get('forwards', [])}

print(f"\n24-25 Only ({len(fwd_24 - fwd_25)}):")
for p in sorted(fwd_24 - fwd_25):
    print(f"  - {p}")

print(f"\n25-26 New ({len(fwd_25 - fwd_24)}):")
for p in sorted(fwd_25 - fwd_24):
    print(f"  + {p}")

print(f"\nCommon ({len(fwd_24 & fwd_25)}):")
for p in sorted(list(fwd_24 & fwd_25)[:5]):
    print(f"    {p}")
print(f"    ... and {len(fwd_24 & fwd_25) - 5} more")

# Goalies
print("\n\nGOALIES:")
g_24 = {f"{p['firstName']['default']} {p['lastName']['default']}" for p in season_24.get('goalies', [])}
g_25 = {f"{p['firstName']['default']} {p['lastName']['default']}" for p in season_25.get('goalies', [])}

print(f"\n24-25 Only:")
for p in sorted(g_24 - g_25):
    print(f"  - {p}")

print(f"\n25-26 New:")
for p in sorted(g_25 - g_24):
    print(f"  + {p}")

print(f"\nCommon:")
for p in sorted(g_24 & g_25):
    print(f"    {p}")
