# RFC-0005 Managed attributes on Image Automation

**Status:** provisional

**Creation date:** 2021-12-16

**Last update:** 2021-12-16

## Summary

Image automation controller can update some attributes of a kubernetes object. 
Today this is limited to image name, image tag and image name+tag. 
This RFC aims to extend this functionality to other attributes.  

## Motivation

Some automation or observability tools can use label to identify better a 
kubernetes object. It can be linked to a version, to a date, to a code 
origin... For multiple reason, the image tag can reflect poorly this
data. An example can be given by the image reflector controller which 
can extract a part of the tag and use it to sort and select the correct one.

### Goals

This RFC aims to describe a way to extract such additional value from the
image tag, and to use them to update some attributes on the kubernetes object.

### Non-Goals

This RFC will focus on image automation controller. It is a non goal to extend
this to manually modified kubernetes objects.

## Proposal

### User Stories

As a user, I can update the filter pattern on the image policy object to 
capture additional data. 
Then, I can reference the name of the captured group in the comment of a 
kubernetes object so that the attribute linked to this comment can be updated. 


### Alternatives

An alternative would be to build a mutation web hook which would be able to 
filter all object and interact with them directly. It would be more generic 
but heavier to build. 
This raise the question on should this be included in flux or not.

## Design Details

Simple update on the image automation controller should be enough. Today a 
filter in the image policy is like: 

```yaml
    extract: $ts
    pattern: ^pr-(?P<pr>.*)-(?P<ts>\d*)-(?P<sha1>.*)$
```

It is possible to modify the image automation to take comment like:

```yaml
  # {"$imagepolicy": "{namespace}:{imagepolicy}:{attributes}"
```

with `attributes` a name of a capture group on the pattern. 

From previous pattern example, accepted attributes will be:

- pr
- ts
- sha1

If a user try to use an attribute name like `tag` or `name` which is 
already defined by flux core, then the original meaning will still be kept :

- tag: the full tag string
- name: the image name

## Implementation History

_not implemented yet_