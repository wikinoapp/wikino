# typed: strict
# frozen_string_literal: true

class NonotoSchema < GraphQL::Schema
  extend T::Sig

  description "A schema of Nonoto."

  mutation Types::Objects::MutationType
  query Types::Objects::QueryType

  use GraphQL::Batch

  sig { params(object: ApplicationRecord, _type: T.untyped, _ctx: T.untyped).returns(String) }
  def self.id_from_object(object, _type = nil, _ctx = nil)
    object.to_gid_param
  end

  sig { params(node_id: String, ctx: GraphQL::Query::Context).returns(T.nilable(ApplicationRecord)) }
  def self.object_from_id(node_id, ctx)
    object = GlobalID.find(node_id)
    object if object.visible_in_graphql?(ctx:)
  end
end
