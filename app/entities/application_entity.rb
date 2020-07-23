# frozen_string_literal: true

class ApplicationEntity < Dry::Struct
  schema schema.strict

  module Types
    include Dry.Types(default: :strict)
  end

  def self.from_nodes(nodes)
    nodes.map do |node|
      from_node(node)
    end
  end
end
