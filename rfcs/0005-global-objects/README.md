# RFC-0005 Memorandum on Flux Global Objects

## Summary

This RFC describes in detail, the need for the concept of global objects.

## Motivation

In multi-tenant environments, especially larger ones, the need for sharing objects across namespaces
increases for several reasons, including security, scalability and complexity.

### Goals

* Establish a common optional 'Global' field on objects that allows the owner to share it with every other
  tenant in a cluster
* Replace 'LocalObjectReference' with 'NamespaceObjectReference' in CR refs, with the required safeguards
  to ensure cross namespace refs are not used for non-'Global' objects.

### Non-Goals

- This is not intended at defining a comprehensive definition of granular authorization for objects


## Security implications

- Cluster admins (and in some cases tenants) might need to share some of the resources they manage
  to other tenants in the cluster. Since many of the types of resources that flux manage are bound
  to namespace local secrets (such as Providers, Gitrepositories, etc), it's not possible to share 
  access to one of these resources without sharing the sensitive secrets they require.

- Namespace isolation is currently available in controllers via `--no-cross-namespace-refs`, and 
  while disabling these by default is good for overall security, it should be possible to override
  this behavior on a specific object's scope, where the owner of the object which is made globally
  available is responsible for the security considerations of exposing their object to other tenants.
  This flag also makes little difference to refs in many object types which right now only take in 
  local refs.

- No RBAC should be required for this, since normally the controllers themselves already have full
  access to every object.

## Kubernetes API usage

- In large scale multi-tenant environments, not being able to share specific resources with multiple
  tenants results in the need to create a (potentially) very large amount of duplicate objects with
  the same information, resulting in additional unnecessary storage in Kubernetes, as well as an 
  increase in API calls

## Flux's authorization model

- While having a fine grained authorization model for flux objects would be ideal, the previously
  discussed requirements for it (ReferenceGrant) might take a very long time to be available and
  does not necessarily conflict with the concept of Global objects, which should be a simple toggle
  and easier for object owners to configure without requiring a long spec.

#### Backwards compatibility

As the NamespaceObjectReference is an extension of LocalObjectReference, backward compatibility is unaffected

