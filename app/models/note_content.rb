# typed: strict
# frozen_string_literal: true

class NoteContent < ApplicationRecord
  belongs_to :user
  belongs_to :note
end
