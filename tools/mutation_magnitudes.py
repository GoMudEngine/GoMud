import glob, re, collections

buckets = collections.defaultdict(list)
for f in glob.glob("_datafiles/world/dogmud/mutations/*.yaml"):
    t = None
    for ln in open(f, encoding="utf-8").read().split("\n"):
        m = re.match(r'\s*-?\s*type:\s*([a-z_]+)', ln)
        if m:
            t = m.group(1)
            continue
        v = re.match(r'\s*value:\s*(-?[0-9.]+)', ln)
        if v and t:
            buckets[t].append(float(v.group(1)))

for t in sorted(buckets):
    vs = buckets[t]
    print(f"{t:30} n={len(vs):3} min={min(vs):8.2f} max={max(vs):8.2f}  vals={sorted(set(vs))}")
