# typed: strict
# frozen_string_literal: true

class NonotoSchema < GraphQL::Schema
  extend T::Sig

  description "A schema of Nonoto."

  mutation Types::Objects::MutationType
  query Types::Objects::QueryType

  use GraphQL::Batch

  sig { params(object: ApplicationRecord, _type: T.untyped, _ctx: T.nilable(GraphQL::Query::Context)).returns(String) }
  def self.id_from_object(object, _type = nil, _ctx = nil)
    object.to_gid_param
  end

  sig { params(node_id: String, _ctx: T.nilable(GraphQL::Query::Context)).returns(T.nilable(ApplicationRecord)) }
  def self.object_from_id(node_id, _ctx = nil)
    GlobalID.find(node_id)
  end
end
