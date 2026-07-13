#!/usr/bin/env python3
"""
Generate a benchmark report with charts from benchmark JSON results.
"""

import json
import os
import sys
from datetime import datetime

import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import matplotlib.ticker as mticker
import numpy as np

# ?? Configuration ??????????????????????????????????????????????????????????
RESULTS_FILE = os.path.join(os.environ.get('TEMP', '.'), 'bench_results.json')
REPORT_DIR = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'report')
CHART_DIR = os.path.join(REPORT_DIR, 'charts')

os.makedirs(CHART_DIR, exist_ok=True)

# ?? Load data ??????????????????????????????????????????????????????????????
# Try to read the JSON results file (may be UTF-8 or UTF-16-LE)
raw = None
for enc in ['utf-8-sig', 'utf-8', 'utf-16-le', 'utf-16']:
    try:
        with open(RESULTS_FILE, 'r', encoding=enc) as f:
            raw = f.read()
        break
    except (UnicodeDecodeError, UnicodeError):
        continue

if raw is None:
    print("Error: Could not read results file with any encoding", file=sys.stderr)
    sys.exit(1)

# Extract JSON array from the output (strip leading text and trailing comparison)
# Find the JSON array that actually contains benchmark results: [{\n    "name":
lines = raw.split('\n')
json_start = -1
json_end = -1
for i, line in enumerate(lines):
    stripped = line.strip()
    if stripped == '[' and json_start < 0:
        # Check if next non-empty line starts with {
        for j in range(i+1, min(i+5, len(lines))):
            if lines[j].strip():
                if lines[j].strip().startswith('{'):
                    json_start = i
                break
    if json_start >= 0 and stripped == ']':
        json_end = i  # Keep looking for the last ']'

if json_start >= 0 and json_end > json_start:
    raw = '\n'.join(lines[json_start:json_end+1])

results = json.loads(raw)

# Group by benchmark name
groups = {}
for r in results:
    name = r['name']
    groups.setdefault(name, []).append(r)

# Color palette
COLORS = {
    'GTSDB': '#4CAF50',
    'InfluxDB': '#2196F3',
    'VM': '#FF9800',
    'NSQ': '#9C27B0',
}

plt.rcParams.update({
    'figure.facecolor': '#1a1a2e',
    'axes.facecolor': '#16213e',
    'axes.edgecolor': '#e0e0e0',
    'axes.labelcolor': '#e0e0e0',
    'axes.titleweight': 'bold',
    'text.color': '#e0e0e0',
    'xtick.color': '#e0e0e0',
    'ytick.color': '#e0e0e0',
    'grid.color': '#2a2a4a',
    'grid.alpha': 0.6,
    'legend.facecolor': '#1a1a2e',
    'legend.edgecolor': '#333',
    'legend.labelcolor': '#e0e0e0',
})

def format_duration(d):
    """Format duration string nicely: 1.234ms -> 1.23 ms, 519us -> 519 us, 1.5s -> 1.50 s."""
    if not d or d == '0s':
        return '0 s'
    # Strip any non-ASCII suffix (handles corrupted micro sign from encoding issues)
    suffix = ''
    for i, ch in enumerate(d):
        if ord(ch) > 127:
            suffix = d[i:]
            d = d[:i]
            break
    if suffix:
        val = float(d)
        return f"{val:.0f} us"
    if 'us' in d:
        return d.replace('us', ' us')
    if 'ns' in d:
        return d.replace('ns', ' ns')
    if 'ms' in d:
        return d.replace('ms', ' ms')
    if d.endswith('s') and 'm' not in d:
        try:
            val = float(d.rstrip('s'))
            return f"{val:.2f} s"
        except ValueError:
            return d
    return d

def parse_duration_to_ms(d):
    """Parse duration string to milliseconds. Handles us, ms, s."""
    if not d or d == '0s':
        return 0
    # Strip any non-ASCII suffix (handles corrupted micro sign)
    suffix = ''
    for i, ch in enumerate(d):
        if ord(ch) > 127:
            suffix = d[i:]
            d = d[:i]
            break
    if suffix:
        # Assume microseconds if there's a non-ASCII suffix
        return float(d) / 1000
    if 'us' in d:
        return float(d.replace('us', '')) / 1000
    if 'ns' in d:
        return float(d.replace('ns', '')) / 1_000_000
    if 'ms' in d:
        return float(d.replace('ms', ''))
    if d.endswith('s') and 'm' not in d:
        return float(d.rstrip('s')) * 1000
    try:
        return float(d.replace('s', '')) * 1000
    except ValueError:
        return 0

def save_chart(fig, name):
    path = os.path.join(CHART_DIR, name)
    fig.savefig(path, dpi=150, bbox_inches='tight', facecolor='#1a1a2e')
    plt.close(fig)
    return f'charts/{name}'

# ??????????????????????????????????????????????????????????????????????????????
# Chart 1: Ops/sec comparison per benchmark (grouped bar chart)
# ??????????????????????????????????????????????????????????????????????????????
def chart_ops_per_sec():
    fig, ax = plt.subplots(figsize=(14, 6))

    bench_names = []
    drivers_data = {}  # driver -> list of (bench_name, ops_sec)

    for r in results:
        name = r['name']
        driver = r['driver']
        ops = r['ops_per_sec']
        bench_names.append(name)
        drivers_data.setdefault(driver, []).append((name, ops))

    bench_names = sorted(set(bench_names))
    # Order: write, read, batch, multi, pipeline, readmany, multiread, multiwrite, pubsub
    order = ['Write (seq)', 'Read (single)', 'Batch Write', 'Multi-Key Write', 'Pipeline Write', 'Multi-Key Read', 'Pub/Sub']
    bench_names = [b for b in order if b in bench_names]

    x = np.arange(len(bench_names))
    width = 0.22
    gap = 0.04

    for i, (driver, color) in enumerate(COLORS.items()):
        vals = []
        for bn in bench_names:
            found = [v for n, v in drivers_data.get(driver, []) if n == bn]
            vals.append(found[0] if found else 0)
        offset = (i - 1.5) * (width + gap)
        bars = ax.bar(x + offset, vals, width, label=driver, color=color, edgecolor='white', linewidth=0.5)
        for bar, v in zip(bars, vals):
            if v > 0:
                ax.text(bar.get_x() + bar.get_width() / 2, bar.get_height(),
                        f'{v:,.0f}', ha='center', va='bottom', fontsize=7, fontweight='bold',
                        color='#e0e0e0')

    ax.set_xticks(x)
    ax.set_xticklabels([b.upper() for b in bench_names], fontsize=11, fontweight='bold')
    ax.set_ylabel('Operations / Second', fontsize=12, fontweight='bold')
    ax.set_title('Throughput Comparison (Higher is Better)', fontsize=14, fontweight='bold', color='#ffffff')
    ax.legend(fontsize=10, loc='upper right')
    ax.set_yscale('log')
    ax.yaxis.set_major_formatter(mticker.FuncFormatter(lambda v, _: f'{v:,.0f}'))
    ax.grid(axis='y', alpha=0.3)

    return save_chart(fig, 'ops_per_sec.png')


# ??????????????????????????????????????????????????????????????????????????????
# Chart 2: Write latency comparison (grouped by driver)
# ??????????????????????????????????????????????????????????????????????????????
def chart_write_latency():
    write_results = [r for r in results if r['name'] == 'Write (seq)' and r['driver'] != 'NSQ']
    if not write_results:
        return None

    fig, ax = plt.subplots(figsize=(10, 5))

    drivers = [r['driver'] for r in write_results]
    means = [parse_duration_to_ms(r['mean']) for r in write_results]
    stds = [parse_duration_to_ms(r['stddev']) for r in write_results]
    colors = [COLORS.get(d, '#888') for d in drivers]

    bars = ax.barh(drivers, means, xerr=stds, color=colors, edgecolor='white', linewidth=0.5,
                   capsize=5, height=0.5)

    for bar, mean, std in zip(bars, means, stds):
        ax.text(bar.get_width() + std + 5, bar.get_y() + bar.get_height() / 2,
                f'{mean:.1f} ± {std:.1f} ms', va='center', fontsize=10, fontweight='bold', color='#e0e0e0')

    ax.set_xlabel('Mean Latency (ms)', fontsize=12, fontweight='bold')
    ax.set_title('Sequential Write Latency (Lower is Better)', fontsize=14, fontweight='bold', color='#ffffff')
    ax.grid(axis='x', alpha=0.3)

    return save_chart(fig, 'write_latency.png')


# ??????????????????????????????????????????????????????????????????????????????
# Chart 3: Batch write ops/sec
# ??????????????????????????????????????????????????????????????????????????????
def chart_batch_comparison():
    batch_results = [r for r in results if r['name'] == 'Batch Write']
    if not batch_results:
        return None

    fig, axes = plt.subplots(1, 2, figsize=(14, 5))

    # Left: ops/sec
    ax = axes[0]
    drivers = [r['driver'] for r in batch_results]
    ops = [r['ops_per_sec'] for r in batch_results]
    colors = [COLORS.get(d, '#888') for d in drivers]
    bars = ax.bar(drivers, ops, color=colors, edgecolor='white', linewidth=0.5, width=0.5)
    for bar, v in zip(bars, ops):
        ax.text(bar.get_x() + bar.get_width() / 2, bar.get_height(),
                f'{v:,.0f}', ha='center', va='bottom', fontsize=9, fontweight='bold', color='#e0e0e0')
    ax.set_title('Batch Write Throughput', fontsize=13, fontweight='bold', color='#ffffff')
    ax.set_ylabel('Ops/sec', fontsize=11, fontweight='bold')
    ax.set_yscale('log')
    ax.yaxis.set_major_formatter(mticker.FuncFormatter(lambda v, _: f'{v:,.0f}'))
    ax.grid(axis='y', alpha=0.3)

    # Right: latency
    ax = axes[1]
    means = [parse_duration_to_ms(r['mean']) for r in batch_results]
    stds_ms = []
    for r in batch_results:
        s = r['stddev']
        stds_ms.append(parse_duration_to_ms(s))
    bars = ax.barh(drivers, means, xerr=stds_ms, color=colors, edgecolor='white', linewidth=0.5,
                   capsize=5, height=0.5)
    for bar, mean in zip(bars, means):
        ax.text(bar.get_width() + 1, bar.get_y() + bar.get_height() / 2,
                f'{mean:.1f} ms', va='center', fontsize=9, fontweight='bold', color='#e0e0e0')
    ax.set_title('Batch Write Latency', fontsize=13, fontweight='bold', color='#ffffff')
    ax.set_xlabel('Latency (ms)', fontsize=11, fontweight='bold')
    ax.grid(axis='x', alpha=0.3)

    plt.tight_layout()
    return save_chart(fig, 'batch_comparison.png')


# ??????????????????????????????????????????????????????????????????????????????
# Chart 4: Multi-sensor write comparison
# ??????????????????????????????????????????????????????????????????????????????
def chart_multi_write():
    multi_results = [r for r in results if r['name'] == 'Multi-Key Write']
    if not multi_results:
        return None

    fig, ax = plt.subplots(figsize=(10, 5))

    drivers = [r['driver'] for r in multi_results]
    ops = [r['ops_per_sec'] for r in multi_results]
    colors = [COLORS.get(d, '#888') for d in drivers]

    bars = ax.bar(drivers, ops, color=colors, edgecolor='white', linewidth=0.5, width=0.5)
    for bar, v in zip(bars, ops):
        ax.text(bar.get_x() + bar.get_width() / 2, bar.get_height(),
                f'{v:,.0f}', ha='center', va='bottom', fontsize=10, fontweight='bold', color='#e0e0e0')

    ax.set_ylabel('Ops/sec', fontsize=12, fontweight='bold')
    ax.set_title('Multi-Sensor Concurrent Write Throughput (Higher is Better)', fontsize=13, fontweight='bold', color='#ffffff')
    ax.set_yscale('log')
    ax.yaxis.set_major_formatter(mticker.FuncFormatter(lambda v, _: f'{v:,.0f}'))
    ax.grid(axis='y', alpha=0.3)

    return save_chart(fig, 'multi_write.png')


# ??????????????????????????????????????????????????????????????????????????????
# Chart 5: Read latency comparison
# ??????????????????????????????????????????????????????????????????????????????
def chart_read_latency():
    read_results = [r for r in results if r['name'] == 'Read (single)']
    readmany_results = [r for r in results if r['name'] == 'Multi-Key Read']
    if not read_results and not readmany_results:
        return None

    fig, axes = plt.subplots(1, 2, figsize=(14, 5))

    # Single read
    ax = axes[0]
    for r in (read_results or []):
        mean = parse_duration_to_ms(r['mean'])
        std = parse_duration_to_ms(r['stddev'])
        color = COLORS.get(r['driver'], '#888')
        ax.barh(r['driver'], mean, xerr=std, color=color, edgecolor='white', linewidth=0.5,
                capsize=5, height=0.4)
        ax.text(mean + std + 0.5, 0, f'{mean:.1f} ± {std:.1f} ms',
                va='center', fontsize=9, fontweight='bold', color='#e0e0e0')
    ax.set_title('Single Read Latency', fontsize=13, fontweight='bold', color='#ffffff')
    ax.set_xlabel('Latency (ms)', fontsize=11, fontweight='bold')
    ax.grid(axis='x', alpha=0.3)

    # Read many
    ax = axes[1]
    for r in (readmany_results or []):
        ops = r['ops_per_sec']
        color = COLORS.get(r['driver'], '#888')
        ax.bar(r['driver'], ops, color=color, edgecolor='white', linewidth=0.5, width=0.4)
    ax.set_title('Read-Many Throughput (5000 reads)', fontsize=13, fontweight='bold', color='#ffffff')
    ax.set_ylabel('Ops/sec', fontsize=11, fontweight='bold')
    ax.grid(axis='y', alpha=0.3)

    plt.tight_layout()
    return save_chart(fig, 'read_comparison.png')


# ??????????????????????????????????????????????????????????????????????????????
# Chart 6: PubSub latency comparison
# ??????????????????????????????????????????????????????????????????????????????
def chart_pubsub():
    pubsub_results = [r for r in results if r['name'] == 'Pub/Sub']
    if not pubsub_results:
        return None

    fig, ax = plt.subplots(figsize=(8, 4))

    drivers = [r['driver'] for r in pubsub_results]
    means = [parse_duration_to_ms(r['mean']) for r in pubsub_results]
    colors = [COLORS.get(d, '#888') for d in drivers]

    bars = ax.bar(drivers, means, color=colors, edgecolor='white', linewidth=0.5, width=0.4)
    for bar, v in zip(bars, means):
        ax.text(bar.get_x() + bar.get_width() / 2, bar.get_height(),
                f'{v:.0f} ms', ha='center', va='bottom', fontsize=11, fontweight='bold', color='#e0e0e0')

    ax.set_ylabel('Latency (ms)', fontsize=12, fontweight='bold')
    ax.set_title('PubSub Message Delivery Latency (5000 msg, Lower is Better)', fontsize=13, fontweight='bold', color='#ffffff')
    ax.grid(axis='y', alpha=0.3)

    return save_chart(fig, 'pubsub.png')


# ??????????????????????????????????????????????????????????????????????????????
# Chart 7: Pipeline comparison (GTSDB only)
# ??????????????????????????????????????????????????????????????????????????????
def chart_pipeline():
    pipe = [r for r in results if r['name'] == 'Pipeline Write']
    write_gtsdb = [r for r in results if r['name'] == 'Write (seq)' and r['driver'] == 'GTSDB']
    if not pipe and not write_gtsdb:
        return None

    fig, ax = plt.subplots(figsize=(8, 4))

    labels = []
    vals = []
    colors_list = []
    for r in pipe:
        labels.append(f"Pipeline ({r['driver']})")
        vals.append(r['ops_per_sec'])
        colors_list.append(COLORS.get(r['driver'], '#888'))
    for r in write_gtsdb:
        labels.append(f"Seq Write ({r['driver']})")
        vals.append(r['ops_per_sec'])
        colors_list.append(COLORS.get(r['driver'], '#888'))

    bars = ax.bar(labels, vals, color=colors_list, edgecolor='white', linewidth=0.5, width=0.5)
    for bar, v in zip(bars, vals):
        ax.text(bar.get_x() + bar.get_width() / 2, bar.get_height(),
                f'{v:,.0f}', ha='center', va='bottom', fontsize=10, fontweight='bold', color='#e0e0e0')

    ax.set_ylabel('Ops/sec', fontsize=12, fontweight='bold')
    ax.set_title('Pipeline vs Sequential Write (GTSDB)', fontsize=13, fontweight='bold', color='#ffffff')
    ax.grid(axis='y', alpha=0.3)

    return save_chart(fig, 'pipeline.png')


# ??????????????????????????????????????????????????????????????????????????????
# Chart 8: Resource usage bar chart (memory + disk)
# ??????????????????????????????????????????????????????????????????????????????

def get_resource_usage():
    """Collect memory (RSS), CPU, and disk usage for each database process."""
    import subprocess as _sp
    import urllib.request as _req
    
    # Force VM to flush in-memory data to disk before measuring
    try:
        _req.urlopen("http://localhost:8428/internal/force_merge?partition_prefix=", timeout=10)
    except Exception:
        pass
    
    info = {}
    
    # Map process names to data directories
    procs = {
        'GTSDB': {'exe': 'gtsdb',        'dir': os.path.abspath(os.path.join(os.path.dirname(REPORT_DIR), '..', 'data'))},
        'InfluxDB': {'exe': 'influxd',   'dir': os.path.join(os.environ.get('TEMP',''), 'influxdb-data', 'engine', 'data')},
        'VM': {'exe': 'victoria-metrics-windows-amd64-prod', 'dir': os.path.join(os.environ.get('TEMP',''), 'vm-data')},
        'NSQ': {'exe': 'nsqd',           'dir': ''},
    }
    
    for name, cfg in procs.items():
        try:
            # Get memory (WorkingSet64) and CPU (TotalProcessorTime) via PowerShell
            cmd = f'powershell -c "Get-Process -Name \'{cfg["exe"]}\' -ErrorAction SilentlyContinue | Select WorkingSet64,CPU,Id | ConvertTo-Json"'
            r = _sp.run(cmd, capture_output=True, text=True, shell=True, timeout=5)
            if r.stdout.strip() and r.stdout.strip() != 'null':
                import json as _j
                data = _j.loads(r.stdout)
                if isinstance(data, dict):
                    data = [data]
                total_mb = sum(d['WorkingSet64'] for d in data) / (1024*1024)
                # CPU is a TimeSpan string like "00:00:12.345" or already seconds as float
                cpu_total = 0.0
                for d in data:
                    cpu_val = d.get('CPU', 0)
                    if isinstance(cpu_val, str):
                        # Parse "hh:mm:ss.fff" or "d.hh:mm:ss.fff"
                        if '.' in cpu_val:
                            main, frac = cpu_val.rsplit('.', 1)
                            frac = float('0.' + frac)
                        else:
                            main, frac = cpu_val, 0.0
                        parts = main.split(':')
                        if len(parts) == 3:
                            cpu_total += int(parts[0]) * 3600 + int(parts[1]) * 60 + int(parts[2]) + frac
                        elif len(parts) == 4:
                            cpu_total += int(parts[0]) * 86400 + int(parts[1]) * 3600 + int(parts[2]) * 60 + int(parts[3]) + frac
                    elif isinstance(cpu_val, (int, float)):
                        cpu_total += float(cpu_val)
                info[name] = {'memory_mb': round(total_mb, 1), 'cpu_sec': round(cpu_total, 1)}
        except Exception:
            if name not in info:
                info[name] = {'memory_mb': 0, 'cpu_sec': 0}
        
        if name not in info:
            info[name] = {'memory_mb': 0, 'cpu_sec': 0}
        
        # Get disk usage
        if cfg['dir'] and os.path.exists(cfg['dir']):
            total_size = 0
            for dirpath, _, filenames in os.walk(cfg['dir']):
                for f in filenames:
                    fp = os.path.join(dirpath, f)
                    try:
                        total_size += os.path.getsize(fp)
                    except OSError:
                        pass
            info[name]['disk_kb'] = round(total_size / 1024, 1)
        else:
            info[name]['disk_kb'] = 0
    
    return info


def chart_resource_usage():
    resources = get_resource_usage()
    dbs = ['GTSDB', 'InfluxDB', 'VM', 'NSQ']
    
    fig, axes = plt.subplots(1, 2, figsize=(12, 4.5))
    
    # Left: Memory
    ax = axes[0]
    mems = [resources.get(d, {}).get('memory_mb', 0) for d in dbs]
    colors = [COLORS.get(d, '#888') for d in dbs]
    bars = ax.bar(dbs, mems, color=colors, edgecolor='white', linewidth=0.5, width=0.5)
    for bar, v in zip(bars, mems):
        if v > 0:
            ax.text(bar.get_x() + bar.get_width() / 2, bar.get_height(),
                    f'{v:.1f} MB', ha='center', va='bottom', fontsize=9, fontweight='bold', color='#e0e0e0')
    ax.set_ylabel('RSS Memory (MB)', fontsize=11, fontweight='bold')
    ax.set_title('Memory Usage', fontsize=13, fontweight='bold', color='#ffffff')
    ax.grid(axis='y', alpha=0.3)
    
    # Right: Disk
    ax = axes[1]
    disks = [resources.get(d, {}).get('disk_kb', 0) for d in dbs]
    bars = ax.bar(dbs, disks, color=colors, edgecolor='white', linewidth=0.5, width=0.5)
    for bar, v in zip(bars, disks):
        if v > 0:
            ax.text(bar.get_x() + bar.get_width() / 2, bar.get_height(),
                    f'{v:.1f} KB', ha='center', va='bottom', fontsize=9, fontweight='bold', color='#e0e0e0')
    ax.set_ylabel('Data on Disk (KB)', fontsize=11, fontweight='bold')
    ax.set_title('Disk Usage', fontsize=13, fontweight='bold', color='#ffffff')
    ax.grid(axis='y', alpha=0.3)
    
    plt.tight_layout()
    return save_chart(fig, 'resource_usage.png')


# ??????????????????????????????????????????????????????????????????????????????
# Generate all charts
# ??????????????????????????????????????????????????????????????????????????????
chart_files = {
    'ops_per_sec': chart_ops_per_sec(),
    'write_latency': chart_write_latency(),
    'batch': chart_batch_comparison(),
    'multi': chart_multi_write(),
    'read': chart_read_latency(),
    'Pub/Sub': chart_pubsub(),
    'pipeline': chart_pipeline(),
    'resource': chart_resource_usage(),
}

# ??????????????????????????????????????????????????????????????????????????????
# Generate markdown report
# ??????????????????????????????????????????????????????????????????????????????

def gen_radar_chart():
    """Generate a radar/spider chart comparing all drivers across benchmarks.
    Uses log-scale normalization so differences are visible and proportional.
    Includes RAM and CPU resource axes (inverted: smaller footprint = higher score)."""
    metrics = ['Write (seq)', 'Read (single)', 'Batch Write', 'Multi-Key Write', 'Pipeline Write', 'Multi-Key Read']
    resource_metrics = ['RAM', 'CPU']
    all_axes = metrics + resource_metrics
    drivers_list = ['GTSDB', 'InfluxDB', 'VM']
    
    resources = get_resource_usage()
    
    fig, ax = plt.subplots(figsize=(8, 8), subplot_kw=dict(polar=True))
    
    angles = np.linspace(0, 2 * np.pi, len(all_axes), endpoint=False).tolist()
    angles += angles[:1]
    
    # Pre-compute per-metric log ranges (floor at log10(1)=0, so minimum gets visible score)
    perf_ranges = {}
    for m in metrics:
        vals = [np.log10(max(r['ops_per_sec'], 1)) for r in results if r['name'] == m and r['ops_per_sec'] > 0]
        if vals:
            hi = max(vals)
            span = hi if hi > 0 else 1  # floor = 0 (log10(1)), so range = [0, hi]
            perf_ranges[m] = hi  # store max; floor is always 0
        else:
            perf_ranges[m] = 1
    
    # Pre-compute resource value ranges (lower = better, invert).
    # Scale so best (lowest) = 100%, worst gets headroom for visibility.
    res_ranges = {}
    for m in resource_metrics:
        key = 'memory_mb' if m == 'RAM' else 'cpu_sec'
        vals = [resources.get(d, {}).get(key, 0) for d in drivers_list]
        lo, hi = min(vals), max(vals)
        span = hi - lo if hi > lo else 1
        # Ceiling = max + 20% padding, so worst still gets visible score
        ceiling = hi + span * 0.2
        res_ranges[m] = (lo, ceiling)  # store floor and ceiling
    
    for driver in drivers_list:
        scores = []
        # Performance metrics: higher = better, per-metric normalization (floor=log10(1)=0)
        for m in metrics:
            found = [r['ops_per_sec'] for r in results if r['name'] == m and r['driver'] == driver]
            val = found[0] if found else 1
            log_val = np.log10(max(val, 1))
            hi = perf_ranges[m]
            scores.append((log_val / hi) * 100 if hi > 0 else 0)
        
        # Resource metrics: lower = better, invert with headroom (best=100%)
        for m in resource_metrics:
            key = 'memory_mb' if m == 'RAM' else 'cpu_sec'
            lo, ceiling = res_ranges[m]
            dr_val = resources.get(driver, {}).get(key, 0)
            scores.append(max(0, (ceiling - dr_val) / (ceiling - lo) * 100))
        
        scores += scores[:1]
        
        color = COLORS.get(driver, '#888')
        ax.plot(angles, scores, 'o-', linewidth=2, label=driver, color=color)
        ax.fill(angles, scores, alpha=0.1, color=color)
    
    ax.set_xticks(angles[:-1])
    ax.set_xticklabels([m.upper() for m in all_axes], fontsize=10, fontweight='bold')
    ax.set_ylim(0, 105)
    ax.set_title('Performance + Resource Profile (Log-Normalized)', fontsize=14, fontweight='bold', 
                 color='#ffffff', pad=20)
    ax.legend(loc='upper right', bbox_to_anchor=(1.3, 1.1), fontsize=10)
    ax.grid(True, alpha=0.3)
    
    return save_chart(fig, 'radar.png')


chart_files['radar'] = gen_radar_chart()

# ?? Build Markdown ??????????????????????????????????????????????????????????

def build_markdown():
    now = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
    resources = get_resource_usage()

    def img(path):
        return f'![{path}]({path})'

    lines = []
    lines.append(f'# Time Series Database Benchmark Report')
    lines.append('')
    lines.append(f'**Generated:** {now}')
    lines.append('')
    lines.append('## Overview')
    lines.append('')
    lines.append('This report compares performance of the following time-series databases:')
    lines.append('')
    lines.append('| Database | Version | Description |')
    lines.append('|----------|---------|-------------|')
    lines.append('| **GTSDB** | v1.0 | Custom Go time-series database (Hamster) |')
    lines.append('| **InfluxDB** | v2.9.1 | Purpose-built time-series database |')
    lines.append('| **VictoriaMetrics** | v1.147.0 | High-performance TSDB (Prometheus-compatible) |')
    lines.append('| **NSQ** | v1.3.0 | Distributed messaging platform (pub/sub only) |')
    lines.append('')
    lines.append('### Benchmark Configuration')
    lines.append('')
    lines.append(f'| Parameter | Value |')
    lines.append('|-----------|-------|')
    lines.append(f'| Points per write | 5,000 |')
    lines.append(f'| Sensors (multi-write) | 5 |')
    lines.append(f'| Runs per benchmark | 3 |')
    lines.append(f'| Warmup iterations | 300 |')
    lines.append('')
    # ?? Resource usage chart ??
    resource_path = chart_files.get('resource')
    if resource_path:
        lines.append('### Resource Usage (Post-Benchmark)')
        lines.append('')
        lines.append(img(resource_path))
        lines.append('')
    # ?? Overall throughput chart ??
    lines.append('## Overall Throughput Comparison')
    lines.append('')
    lines.append(img(chart_files['ops_per_sec']))
    lines.append('')
    lines.append('*Log scale. Higher bars indicate better throughput (ops/sec).*')
    lines.append('')

    # ?? Radar chart ??
    lines.append('## Performance Profile (Radar)')
    lines.append('')
    lines.append(img(chart_files['radar']))
    lines.append('')
    lines.append('*Radar chart showing normalized performance across all benchmark types. 100 = best in class.*')
    lines.append('')

    # ?? Write benchmarks ??
    lines.append('## Write Benchmarks')
    lines.append('')

    # Sequential write
    lines.append('### Sequential Single-Point Writes')
    lines.append('')
    lines.append(img(chart_files['write_latency']))
    lines.append('')
    lines.append('| Driver | Mean | StdDev | Min | Max | P50 | P95 | P99 | Ops/sec |')
    lines.append('|--------|------|--------|-----|-----|-----|-----|-----|---------|')
    for r in results:
        if r['name'] == 'Write (seq)':
            lines.append(
                f"| {r['driver']} | {format_duration(r['mean'])} | {format_duration(r['stddev'])} | "
                f"{format_duration(r['min'])} | {format_duration(r['max'])} | "
                f"{format_duration(r['p50'])} | {format_duration(r['p95'])} | {format_duration(r['p99'])} | "
                f"{r['ops_per_sec']:,.1f} |"
            )
    lines.append('')

    # Pipeline write (GTSDB)
    pipe_gtsdb = [r for r in results if r['name'] == 'Pipeline Write']
    if pipe_gtsdb:
        lines.append('### Pipelined Writes (GTSDB)')
        lines.append('')
        lines.append(img(chart_files['pipeline']))
        lines.append('')
        lines.append('| Driver | Mean | StdDev | Min | Max | P50 | P95 | P99 | Ops/sec |')
        lines.append('|--------|------|--------|-----|-----|-----|-----|-----|---------|')
        for r in pipe_gtsdb:
            lines.append(
                f"| {r['driver']} | {format_duration(r['mean'])} | {format_duration(r['stddev'])} | "
                f"{format_duration(r['min'])} | {format_duration(r['max'])} | "
                f"{format_duration(r['p50'])} | {format_duration(r['p95'])} | {format_duration(r['p99'])} | "
                f"{r['ops_per_sec']:,.1f} |"
            )
        lines.append('')

    # ?? Batch write ??
    lines.append('### Batch/Bulk Writes')
    lines.append('')
    lines.append(img(chart_files['batch']))
    lines.append('')
    lines.append('| Driver | Mean | StdDev | Min | Max | P50 | P95 | P99 | Ops/sec |')
    lines.append('|--------|------|--------|-----|-----|-----|-----|-----|---------|')
    for r in results:
        if r['name'] == 'Batch Write':
            lines.append(
                f"| {r['driver']} | {format_duration(r['mean'])} | {format_duration(r['stddev'])} | "
                f"{format_duration(r['min'])} | {format_duration(r['max'])} | "
                f"{format_duration(r['p50'])} | {format_duration(r['p95'])} | {format_duration(r['p99'])} | "
                f"{r['ops_per_sec']:,.1f} |"
            )
    lines.append('')

    # ?? Multi-sensor write ??
    lines.append('### Multi-Sensor Concurrent Writes')
    lines.append('')
    lines.append(img(chart_files['multi']))
    lines.append('')
    lines.append('| Driver | Mean | StdDev | Min | Max | P50 | P95 | P99 | Ops/sec |')
    lines.append('|--------|------|--------|-----|-----|-----|-----|-----|---------|')
    for r in results:
        if r['name'] == 'Multi-Key Write':
            lines.append(
                f"| {r['driver']} | {format_duration(r['mean'])} | {format_duration(r['stddev'])} | "
                f"{format_duration(r['min'])} | {format_duration(r['max'])} | "
                f"{format_duration(r['p50'])} | {format_duration(r['p95'])} | {format_duration(r['p99'])} | "
                f"{r['ops_per_sec']:,.1f} |"
            )
    lines.append('')

    # ?? Read benchmarks ??
    lines.append('## Read Benchmarks')
    lines.append('')
    lines.append(img(chart_files['read']))
    lines.append('')

    # Single read
    lines.append('### Single Read (Last 1 Point)')
    lines.append('')
    lines.append('| Driver | Mean | StdDev | Min | Max | P50 | P95 | P99 | Ops/sec |')
    lines.append('|--------|------|--------|-----|-----|-----|-----|-----|---------|')
    for r in results:
        if r['name'] == 'Read (single)':
            lines.append(
                f"| {r['driver']} | {format_duration(r['mean'])} | {format_duration(r['stddev'])} | "
                f"{format_duration(r['min'])} | {format_duration(r['max'])} | "
                f"{format_duration(r['p50'])} | {format_duration(r['p95'])} | {format_duration(r['p99'])} | "
                f"{r['ops_per_sec']:,.1f} |"
            )
    lines.append('')

    # Read many
    lines.append('### Read-Many (5000 Reads)')
    lines.append('')
    lines.append('| Driver | Mean | StdDev | Min | Max | P50 | P95 | P99 | Ops/sec |')
    lines.append('|--------|------|--------|-----|-----|-----|-----|-----|---------|')
    for r in results:
        if r['name'] == 'Multi-Key Read':
            lines.append(
                f"| {r['driver']} | {format_duration(r['mean'])} | {format_duration(r['stddev'])} | "
                f"{format_duration(r['min'])} | {format_duration(r['max'])} | "
                f"{format_duration(r['p50'])} | {format_duration(r['p95'])} | {format_duration(r['p99'])} | "
                f"{r['ops_per_sec']:,.1f} |"
            )
    lines.append('')

    # ?? PubSub ??
    lines.append('## Pub/Sub Benchmark')
    lines.append('')
    lines.append(img(chart_files['Pub/Sub']))
    lines.append('')
    lines.append('| Driver | Mean | StdDev | Min | Max | P50 | P95 | P99 | Ops/sec |')
    lines.append('|--------|------|--------|-----|-----|-----|-----|-----|---------|')
    for r in results:
        if r['name'] == 'Pub/Sub':
            lines.append(
                f"| {r['driver']} | {format_duration(r['mean'])} | {format_duration(r['stddev'])} | "
                f"{format_duration(r['min'])} | {format_duration(r['max'])} | "
                f"{format_duration(r['p50'])} | {format_duration(r['p95'])} | {format_duration(r['p99'])} | "
                f"{r['ops_per_sec']:,.1f} |"
            )
    lines.append('')

    # ?? Comparison summary ??
    lines.append('## Key Findings')
    lines.append('')
    lines.append('### Sequential Write Throughput')
    lines.append('')
    write_results_sorted = sorted(
        [r for r in results if r['name'] == 'Write (seq)'],
        key=lambda x: x['ops_per_sec'], reverse=True
    )
    for i, r in enumerate(write_results_sorted):
        medal = ['#1', '#2', '#3'][i] if i < 3 else f'#{i+1}'
        lines.append(f"- {medal} **{r['driver']}**: **{r['ops_per_sec']:,.0f} ops/sec** ({format_duration(r['mean'])})")
    lines.append('')

    lines.append('### Batch Write Throughput')
    lines.append('')
    batch_results_sorted = sorted(
        [r for r in results if r['name'] == 'Batch Write'],
        key=lambda x: x['ops_per_sec'], reverse=True
    )
    for i, r in enumerate(batch_results_sorted):
        medal = ['#1', '#2', '#3'][i] if i < 3 else f'#{i+1}'
        lines.append(f"- {medal} **{r['driver']}**: **{r['ops_per_sec']:,.0f} ops/sec** ({format_duration(r['mean'])})")
    lines.append('')

    lines.append('### Multi-Sensor Write Throughput')
    lines.append('')
    multi_results_sorted = sorted(
        [r for r in results if r['name'] == 'Multi-Key Write'],
        key=lambda x: x['ops_per_sec'], reverse=True
    )
    for i, r in enumerate(multi_results_sorted):
        medal = ['#1', '#2', '#3'][i] if i < 3 else f'#{i+1}'
        lines.append(f"- {medal} **{r['driver']}**: **{r['ops_per_sec']:,.0f} ops/sec** ({format_duration(r['mean'])})")
    lines.append('')

    lines.append('### Read Performance')
    lines.append('')
    read_results_sorted = sorted(
        [r for r in results if r['name'] == 'Read (single)'],
        key=lambda x: x['ops_per_sec'], reverse=True
    )
    for i, r in enumerate(read_results_sorted):
        medal = ['#1', '#2', '#3'][i] if i < 3 else f'#{i+1}'
        lines.append(f"- {medal} **{r['driver']}**: **{r['ops_per_sec']:,.0f} ops/sec** ({format_duration(r['mean'])})")
    lines.append('')

    lines.append('### Pub/Sub Latency')
    lines.append('')
    pubsub_results_sorted = sorted(
        [r for r in results if r['name'] == 'Pub/Sub'],
        key=lambda x: parse_duration_to_ms(x['mean'])
    )
    for i, r in enumerate(pubsub_results_sorted):
        medal = ['#1', '#2'][i] if i < 2 else f'#{i+1}'
        lines.append(f"- {medal} **{r['driver']}**: **{format_duration(r['mean'])}** delivery latency")
    lines.append('')

    # ?? Comparison ratios ??
    lines.append('## Head-to-Head Comparison')
    lines.append('')
    lines.append('| Benchmark | Comparison | Ratio | Winner |')
    lines.append('|-----------|------------|-------|--------|')
    seen_pairs = set()
    for r in results:
        for r2 in results:
            if r['name'] == r2['name'] and r['driver'] != r2['driver']:
                pair_key = tuple(sorted([r['driver'], r2['driver']])) + (r['name'],)
                if pair_key in seen_pairs:
                    continue
                seen_pairs.add(pair_key)

                ops1 = r['ops_per_sec']
                ops2 = r2['ops_per_sec']
                if ops1 == 0 or ops2 == 0:
                    continue
                ratio = ops1 / ops2
                if r['name'] in ('Pub/Sub',):
                    # For latencies, lower is better
                    lat1 = parse_duration_to_ms(r['mean'])
                    lat2 = parse_duration_to_ms(r2['mean'])
                    if lat1 == 0 or lat2 == 0:
                        continue
                    ratio = lat2 / lat1
                    winner = r['driver'] if lat1 < lat2 else r2['driver']
                    lines.append(f"| {r['name']} | {r['driver']} vs {r2['driver']} | {ratio:.2f}x latency | **{winner}** |")
                else:
                    winner = r['driver'] if ratio > 1 else r2['driver']
                    ratio_display = ratio if ratio >= 1 else (1/ratio)
                    faster = r['driver'] if ratio > 1 else r2['driver']
                    lines.append(f"| {r['name']} | {r['driver']} vs {r2['driver']} | {ratio_display:.2f}x | **{faster}** |")

    return '\n'.join(lines)


markdown = build_markdown()

# Write report
report_path = os.path.join(REPORT_DIR, 'BENCHMARK_REPORT.md')
with open(report_path, 'w', encoding='utf-8') as f:
    f.write(markdown)

# Write JSON data file for homepage auto-update
def get_mean_ms(name, driver):
    """Get mean duration in milliseconds for a benchmark result."""
    for r in results:
        if r['name'] == name and r['driver'] == driver:
            return parse_duration_to_ms(r['mean'])
    return 0

def get_ratio(name, driver1, driver2):
    """Get speedup ratio (driver1/driver2) for a benchmark.
    For near-zero latencies, uses mean duration comparison."""
    ops1 = next((r['ops_per_sec'] for r in results if r['name'] == name and r['driver'] == driver1), 0)
    ops2 = next((r['ops_per_sec'] for r in results if r['name'] == name and r['driver'] == driver2), 0)
    m1 = next((parse_duration_to_ms(r['mean']) for r in results if r['name'] == name and r['driver'] == driver1), 0)
    m2 = next((parse_duration_to_ms(r['mean']) for r in results if r['name'] == name and r['driver'] == driver2), 0)
    
    # If either ops/sec is 0 (sub-ms measurements), fall back to mean duration
    if ops1 == 0 or ops2 == 0:
        if m1 > 0 and m2 > 0:
            return round(max(m1, m2) / min(m1, m2), 1)
        if m1 > 0 and m2 == 0:
            return round(m1 / 0.05, 1)  # estimate from sub-ms baseline
        if m2 > 0 and m1 == 0:
            return round(m2 / 0.05, 1)
        return 1  # both effectively 0
    
    return round(ops1 / ops2, 2) if ops1 >= ops2 else round(ops2 / ops1, 2)

def get_read_ratio(name, gtsdb_ns, other_ns):
    """Get read speedup ratio. If both near zero, use ops/sec comparison."""
    if gtsdb_ns < 0.1 and other_ns < 0.1:
        # Both sub-ms, use ops/sec ratio
        return get_ratio(name, 'GTSDB', 'VM' if 'VM' in str(other_ns) else 'InfluxDB')
    if gtsdb_ns > 0 and other_ns > 0:
        return round(max(other_ns, gtsdb_ns) / min(other_ns, gtsdb_ns), 1)
    return 99

gtsdb_read_single = get_mean_ms('Read (single)', 'GTSDB')
vm_read_single = get_mean_ms('Read (single)', 'VM')
influx_read_single = get_mean_ms('Read (single)', 'InfluxDB')
gtsdb_read_many = get_mean_ms('Multi-Key Read', 'GTSDB')
vm_read_many = get_mean_ms('Multi-Key Read', 'VM')
influx_read_many = get_mean_ms('Multi-Key Read', 'InfluxDB')

data = {
    "write": {
        "gtsdb": get_mean_ms('Write (seq)', 'GTSDB'),
        "vm": get_mean_ms('Write (seq)', 'VM'),
        "influxdb": get_mean_ms('Write (seq)', 'InfluxDB'),
    },
    "batchWrite": {
        "gtsdb": get_mean_ms('Batch Write', 'GTSDB'),
        "vm": get_mean_ms('Batch Write', 'VM'),
        "influxdb": get_mean_ms('Batch Write', 'InfluxDB'),
    },
    "pipeline": {
        "gtsdb": get_mean_ms('Pipeline Write', 'GTSDB'),
        "vm": get_mean_ms('Pipeline Write', 'VM'),
        "influxdb": get_mean_ms('Pipeline Write', 'InfluxDB'),
    },
    "multiWrite": {
        "gtsdb": get_mean_ms('Multi-Key Write', 'GTSDB'),
        "vm": get_mean_ms('Multi-Key Write', 'VM'),
        "influxdb": get_mean_ms('Multi-Key Write', 'InfluxDB'),
    },
    "read": {
        "gtsdb": max(gtsdb_read_single, 0.05),
        "vm": max(vm_read_single, 0.05),
        "influxdb": max(influx_read_single, 0.05),
    },
    "readMany": {
        "gtsdb": max(gtsdb_read_many, 0.05),
        "vm": max(vm_read_many, 0.05),
        "influxdb": max(influx_read_many, 0.05),
    },
    "pubsub": {
        "gtsdb": get_mean_ms('Pub/Sub', 'GTSDB') / 1000,
        "nsq": get_mean_ms('Pub/Sub', 'NSQ') / 1000,
    },
    "ratios": {
        "writeVsInflux": get_ratio('Write (seq)', 'GTSDB', 'InfluxDB'),
        "writeVsVM": get_ratio('Write (seq)', 'GTSDB', 'VM'),
        "pipelineVsInflux": get_ratio('Pipeline Write', 'GTSDB', 'InfluxDB'),
        "pipelineVsVM": get_ratio('Pipeline Write', 'GTSDB', 'VM'),
        "batchVsInflux": get_ratio('Batch Write', 'GTSDB', 'InfluxDB'),
        "batchVsVM": get_ratio('Batch Write', 'VM', 'GTSDB'),
        "multiWriteVsInflux": get_ratio('Multi-Key Write', 'GTSDB', 'InfluxDB'),
        "multiWriteVsVM": get_ratio('Multi-Key Write', 'VM', 'GTSDB'),
        "readVsInflux": get_ratio('Read (single)', 'GTSDB', 'InfluxDB'),
        "readVsVM": get_ratio('Read (single)', 'GTSDB', 'VM'),
        "readManyVsInflux": get_ratio('Multi-Key Read', 'GTSDB', 'InfluxDB'),
        "readManyVsVM": get_ratio('Multi-Key Read', 'GTSDB', 'VM'),
    },
    "resources": get_resource_usage(),
}

data_path = os.path.join(REPORT_DIR, 'benchmark-data.json')
with open(data_path, 'w', encoding='utf-8') as f:
    json.dump(data, f, indent=2)

import sys as _sys
print(f'[OK] Report generated: {report_path}', file=_sys.stdout)
print(f'[OK] Charts saved to: {CHART_DIR}', file=_sys.stdout)
print(f'[OK] Total benchmarks: {len(results)}', file=_sys.stdout)
