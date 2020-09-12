# frozen_string_literal: true

class LinkEntity < ApplicationEntity
  attribute? :note, NoteEntity

  def self.from_node(node)
    attrs = {}

    if note_node = node["note"]
      attrs[:note] = NoteEntity.from_node(note_node)
    end

    new attrs
  end
end
