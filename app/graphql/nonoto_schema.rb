# frozen_string_literal: true

class NonotoSchema < GraphQL::Schema
  mutation Types::Object::MutationType
  query Types::Object::QueryType

  use GraphQL::Execution::Interpreter
  use GraphQL::Analysis::AST
  use GraphQL::Pagination::Connections

  def self.id_from_object(object, type_definition, query_ctx = nil)
    GraphQL::Schema::UniqueWithinType.encode(type_definition.name, object.id).delete("=")
  end

  def self.object_from_id(id, query_ctx = nil)
    resource_name, resource_id = GraphQL::Schema::UniqueWithinType.decode(id)

    return nil if resource_name.blank? || resource_id.blank?

    Object.const_get(resource_name).find(resource_id)
  end

  def self.resolve_type(_type, obj, _ctx)
    case obj
    when Note
      Types::Object::NoteType
    else
      raise("Unexpected object: #{obj}")
    end
  end
end
