# frozen_string_literal: true

class NoteEntity < ApplicationEntity
  attribute? :database_id, Types::String
  attribute? :title, Types::String
  attribute? :body, Types::String

  def self.from_node(note_node)
    attrs = {}

    if database_id = note_node["databaseId"]
      attrs[:database_id] = database_id
    end

    if title = note_node["title"]
      attrs[:title] = title
    end

    new attrs
  end
end
