# typed: strict
# frozen_string_literal: true

class NoteEditorship < ApplicationRecord
  belongs_to :space
  belongs_to :note
  belongs_to :editor, class_name: "User"
end
