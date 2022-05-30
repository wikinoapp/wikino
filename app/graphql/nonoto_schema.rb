# typed: strict
# frozen_string_literal: true

class NonotoSchema < GraphQL::Schema
  extend T::Sig

  description "A schema of Nonoto."

  mutation Types::Objects::Mutation
  query Types::Objects::Query

  use GraphQL::Batch

  sig { params(object: ApplicationRecord, _type: T.untyped, _ctx: T.nilable(GraphQL::Query::Context)).returns(String) }
  def self.id_from_object(object, _type = nil, _ctx = nil)
    object.to_gid_param
  end

  sig { params(node_id: String, ctx: GraphQL::Query::Context).returns(T.nilable(ApplicationRecord)) }
  def self.object_from_id(node_id, ctx)
    global_id = GlobalID.parse(node_id)
    scope = Pundit.policy_scope(ctx[:viewer], global_id&.model_class)

    scope&.find_by(id: global_id&.model_id)
  end
end
