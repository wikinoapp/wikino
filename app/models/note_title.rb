# typed: strict
# frozen_string_literal: true

class NoteTitle < ApplicationRecord
  belongs_to :space
  belongs_to :list
  belongs_to :note
end
