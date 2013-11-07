# TODO: remove this and update the paths to point to the directories
# once Go CLI is fixed to preserve executable permissions of bin/*
# 
# see https://www.pivotaltracker.com/story/show/60124570

all: assets/buildpacks/simple-buildpack.zip assets/buildpacks/another-buildpack.zip

assets/buildpacks/simple-buildpack.zip: assets/buildpacks/simple-buildpack/bin/*
	cd assets/buildpacks/simple-buildpack && \
	  zip -r simple-buildpack.zip bin && \
	  mv simple-buildpack.zip ../

assets/buildpacks/another-buildpack.zip: assets/buildpacks/simple-buildpack/bin/*
	cd assets/buildpacks/another-buildpack && \
	  zip -r another-buildpack.zip bin && \
	  mv another-buildpack.zip ../
