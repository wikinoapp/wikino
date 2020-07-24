# frozen_string_literal: true

class MutationErrorEntity < ApplicationEntity
  attribute :message, Types::String

  def self.from_node(mutation_error_node)
    attrs = {}

    attrs[:message] = mutation_error_node["message"]

    new attrs
  end
end
