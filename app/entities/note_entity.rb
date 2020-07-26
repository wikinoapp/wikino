# frozen_string_literal: true

class NoteEntity < ApplicationEntity
  attribute? :database_id, Types::String
  attribute? :name, Types::String
  attribute? :body, Types::String

  def self.from_node(note_node)
    attrs = {}

    if database_id = note_node["databaseId"]
      attrs[:database_id] = database_id
    end

    if name = note_node["name"]
      attrs[:name] = name
    end

    if body = note_node["body"]
      attrs[:body] = body
    end

    new attrs
  end
end
