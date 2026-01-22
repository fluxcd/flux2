# RFC-0005 Extend supported list of image automation marker reference attributes

**Status:** provisional

**Creation date:** 2021-12-16

**Last update:** 2022-01-25

## Summary

Flux should allow referencing more metadata in the image automation Setters strategy.

## Motivation

Some automation or observability tools can use label to identify better a 
kubernetes object. It can be linked to a version, a date, a code 
origin... For multiple reason, the image tag can reflect poorly this
data. An example can be given by the image reflector controller which 
can extract a part of the tag and use it to sort and select the correct one.

### Goals

This RFC aims to describe

- A way to extract such additional attributes from the image tag.
- Use those new attributes to update the kubernetes object.

### Non-Goals

This RFC will focus on Image Automation Controller and Image Reflector Controller.

It is a non goal to keep in sync the attributes if the kubernetes object is 
updated manually.

## Proposal

### User Stories

As a user, I can update the filter pattern on the image policy object to 
capture additional data. 
Then, I can reference the name of the captured group in the comment of a 
kubernetes object so that the attribute linked to this comment can be updated. 


### Alternatives

An alternative would be to build a mutation web hook which would be able to 
filter all object and interact with them directly. 

It would be more generic, more customizable and safer (fix the manual update use case)
to create such mutation web hook, but will be more complex to build. 
(new kubernetes object, new controller)  

This raise the question on should this feature to be included in flux or not. 

## Design Details

Two options are possible here:

- Only modify the Image Automation Controller to make it read ImagePolicies spec
and compute attributes
- Modify the Image Reflector Controller, to extract the attributes, stores them
in the status and update the Image Automation Controller to use this new data storage. 

The second option seems to be preferable to separate concerns.
 
A simple option would be to allow multiple capture group in the filter in the ImagePolicy: 

```yaml
    extract: $ts
    pattern: ^pr-(?P<pr>.*)-(?P<ts>\d*)-(?P<sha1>.*)$
```

And then to modify the Image Automation Controller to take comment like:

```yaml
  # {"$imagepolicy": "{namespace}:{imagepolicy}:{attributes}"
```

with `attributes` a name of a capture group on the pattern. 

From previous pattern example, accepted attributes will be:

- pr
- ts
- sha1

If a user try to capture an attribute with a name like `tag` or `name` (already defined 
by flux core), then the original value will be kept and a warning should show on the 
Image Reflector Controller logs.

As reminder, here is the definition for those default attributes:

- tag: the full tag string
- name: the image name

## Implementation History

_not implemented yet_