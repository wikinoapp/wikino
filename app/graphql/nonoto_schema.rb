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
    return unless query_ctx

    begin
      resource_type, resource_id = GraphQL::Schema::UniqueWithinType.decode(id)
    rescue ArgumentError => e
      Rails.logger.warn "Could not decode data: #{e}, #{id}"
    end
    return if resource_type.blank? || resource_id.blank?

    user_resources(query_ctx[:viewer], resource_type).find(resource_id)
  end

  def self.resolve_type(_type, obj, _ctx)
    case obj
    when Note
      Types::Object::NoteType
    else
      raise "Unexpected object: #{obj}"
    end
  end

  class << self
    private

    def user_resources(user, resource_type)
      case resource_type
      when "Note"
        user.notes
      else
        raise "Unexpected resource_type: #{resource_type}"
      end
    end
  end
end
