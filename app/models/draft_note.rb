# typed: strict
# frozen_string_literal: true

class DraftNote < ApplicationRecord
  include ModelConcerns::NoteEditable

  belongs_to :space
  belongs_to :notebook
  belongs_to :note
  belongs_to :editor, class_name: "User"
end
