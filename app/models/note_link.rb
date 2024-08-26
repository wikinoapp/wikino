# typed: strict
# frozen_string_literal: true

class NoteLink < ApplicationRecord
  belongs_to :space
  belongs_to :source_note, class_name: "Note"
  belongs_to :target_note, class_name: "Note"
end
