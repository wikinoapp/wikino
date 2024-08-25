# typed: strict
# frozen_string_literal: true

class NoteRevision < ApplicationRecord
  belongs_to :space
  belongs_to :editor, class_name: "User"
  belongs_to :note
end
