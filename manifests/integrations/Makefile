
bases := $(shell dirname $(shell find | grep kustomization.yaml | sort))

all: $(bases)

permutations := $(bases) $(addsuffix /,$(bases))
.PHONY: $(permutations)
$(permutations):
	@echo $@
	@warnings=$$(kustomize build $@ -o /dev/null 2>&1); \
		if [ "$$warnings" ]; then \
			echo "$$warnings"; \
			false; \
		fi
