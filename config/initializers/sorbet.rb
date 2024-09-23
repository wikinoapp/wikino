# typed: strict
# frozen_string_literal: true

# https://github.com/Shopify/tapioca/issues/1140#issuecomment-1233158782
ActiveRecord::Base.descendants.each do |klass|
  klass.const_set(:PrivateAssociationRelation, Object)
  klass.const_set(:PrivateAssociationRelationWhereChain, Object)
  klass.const_set(:PrivateCollectionProxy, Object)
  klass.const_set(:PrivateRelation, Object)
  klass.const_set(:PrivateRelationWhereChain, Object)
end
