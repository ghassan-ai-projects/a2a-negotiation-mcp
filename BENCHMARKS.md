goos: darwin
goarch: arm64
pkg: github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/server
cpu: Apple M1 Pro
BenchmarkPricingQuery-10      	      30	   3787017 ns/op	  691278 B/op	   16102 allocs/op
BenchmarkCreateSession-10     	    2936	     40725 ns/op	    5584 B/op	     126 allocs/op
BenchmarkRunNegotiation-10    	    1156	    127186 ns/op	   13350 B/op	     319 allocs/op
BenchmarkParallel2-10         	     482	    265537 ns/op	   26945 B/op	     648 allocs/op
BenchmarkParallel5-10         	     136	    871445 ns/op	   98624 B/op	    1945 allocs/op
BenchmarkComputeOffer-10      	    2056	     58215 ns/op	   13644 B/op	     334 allocs/op
BenchmarkQuoteParse-10        	    2047	     57283 ns/op	    5852 B/op	     125 allocs/op
BenchmarkContractParse-10     	     392	    303381 ns/op	    4676 B/op	      44 allocs/op
PASS
ok  	github.com/ghassan-ai-projects/a2a-negotiation-mcp/internal/server	1.633s
