# prometheus_turbotstat_exporter
An prometheus exporter for turbotstat (for monitoring different C-states and pkg-states).

## Example scrape output

Part of the output from the scrape:

```txt
...
turbostat_cpu_states_percent{num_cpu="6",type="c1e"} 0.03
turbostat_cpu_states_percent{num_cpu="6",type="c3"} 0.05
turbostat_cpu_states_percent{num_cpu="6",type="c6"} 1.37
turbostat_cpu_states_percent{num_cpu="6",type="c7s"} 0
turbostat_cpu_states_percent{num_cpu="6",type="c8"} 1.11
turbostat_cpu_states_percent{num_cpu="6",type="c9"} 0.13
turbostat_cpu_states_percent{num_cpu="6",type="poll"} 0
turbostat_cpu_states_percent{num_cpu="7",type="c1"} 0.01
turbostat_cpu_states_percent{num_cpu="7",type="c10"} 96.91
turbostat_cpu_states_percent{num_cpu="7",type="c1e"} 0.03
turbostat_cpu_states_percent{num_cpu="7",type="c3"} 0.01
turbostat_cpu_states_percent{num_cpu="7",type="c6"} 1.36
...
```
