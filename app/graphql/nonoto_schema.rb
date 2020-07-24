# frozen_string_literal: true

class NonotoSchema < GraphQL::Schema
  mutation Types::Objects::MutationType
  query Types::Objects::QueryType

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

    scoped_resources(resource_type, query_ctx).find(resource_id)
  end

  def self.resolve_type(_type, obj, _ctx)
    case obj
    when Note
      Types::Objects::NoteType
    else
      raise "Unexpected object: #{obj}"
    end
  end

  class << self
    private

    def scoped_resources(resource_type, query_ctx = nil)
      return unless query_ctx

      user = query_ctx[:viewer]

      case resource_type
      when "Note"
        user.notes
      else
        raise "Unexpected resource_type: #{resource_type}"
      end
    end
  end
end
