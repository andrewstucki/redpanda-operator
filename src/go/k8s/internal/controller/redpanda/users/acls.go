package users

import (
	redpandav1alpha2 "github.com/redpanda-data/redpanda-operator/src/go/k8s/api/redpanda/v1alpha2"
	"github.com/twmb/franz-go/pkg/kmsg"
)

type aclRule struct {
	ResourceType        kmsg.ACLResourceType
	ResourceName        string
	ResourcePatternType kmsg.ACLResourcePatternType
	Principal           string
	Host                string
	Operation           kmsg.ACLOperation
	PermissionType      kmsg.ACLPermissionType
}

func (r aclRule) toDeletionFilter() kmsg.DeleteACLsRequestFilter {
	return kmsg.DeleteACLsRequestFilter{
		ResourceType:        r.ResourceType,
		ResourceName:        &r.ResourceName,
		ResourcePatternType: r.ResourcePatternType,
		Principal:           &r.Principal,
		Host:                &r.Host,
		Operation:           r.Operation,
		PermissionType:      r.PermissionType,
	}
}

func aclRuleFromCreation(creation kmsg.CreateACLsRequestCreation) aclRule {
	return aclRule{
		ResourceType:        creation.ResourceType,
		ResourceName:        creation.ResourceName,
		ResourcePatternType: creation.ResourcePatternType,
		Principal:           creation.Principal,
		Host:                creation.Host,
		Operation:           creation.Operation,
		PermissionType:      creation.PermissionType,
	}
}

func aclRuleSetFromACLs(acls *kmsg.DescribeACLsResponse) map[aclRule]struct{} {
	rules := map[aclRule]struct{}{}

	if acls == nil {
		return rules
	}

	for _, acls := range acls.Resources {
		for _, acl := range acls.ACLs {
			rules[aclRule{
				ResourceType:        acls.ResourceType,
				ResourceName:        acls.ResourceName,
				ResourcePatternType: acls.ResourcePatternType,
				Principal:           acl.Principal,
				Host:                acl.Host,
				Operation:           acl.Operation,
				PermissionType:      acl.PermissionType,
			}] = struct{}{}
		}
	}

	return rules
}

func calculateACLs(user *redpandav1alpha2.User, acls *kmsg.DescribeACLsResponse) ([]kmsg.CreateACLsRequestCreation, []kmsg.DeleteACLsRequestFilter, error) {
	existing := aclRuleSetFromACLs(acls)
	toDelete := aclRuleSetFromACLs(acls)

	creations := map[aclRule]kmsg.CreateACLsRequestCreation{}

	for _, rule := range user.Spec.Authorization.ACLs {
		resourceType, err := kmsg.ParseACLResourceType(rule.Resource.Type)
		if err != nil {
			return nil, nil, err
		}

		permType, err := kmsg.ParseACLPermissionType(rule.Type)
		if err != nil {
			return nil, nil, err
		}

		patternType, err := kmsg.ParseACLResourcePatternType(rule.Resource.PatternType)
		if err != nil {
			return nil, nil, err
		}

		for _, operation := range rule.Operations {
			op, err := kmsg.ParseACLOperation(operation)
			if err != nil {
				return nil, nil, err
			}

			var creation kmsg.CreateACLsRequestCreation

			creation.ResourceType = resourceType
			creation.Host = rule.Host
			if creation.Host == "" {
				creation.Host = "*"
			}
			creation.PermissionType = permType
			creation.ResourcePatternType = patternType
			creation.ResourceName = rule.Resource.Name
			creation.Principal = "User:" + user.RedpandaName()
			creation.Operation = op

			aclRule := aclRuleFromCreation(creation)
			delete(toDelete, aclRule)

			if _, exists := existing[aclRule]; exists {
				continue
			}

			creations[aclRule] = creation
		}
	}

	var flattenedCreations []kmsg.CreateACLsRequestCreation
	for _, creation := range creations {
		flattenedCreations = append(flattenedCreations, creation)
	}

	var deletions []kmsg.DeleteACLsRequestFilter
	for deletion := range toDelete {
		deletions = append(deletions, deletion.toDeletionFilter())
	}

	return flattenedCreations, deletions, nil
}
