# typed: strict
# frozen_string_literal: true

class NoteEditor < ApplicationRecord
  extend T::Sig

  belongs_to :space
  belongs_to :note
  belongs_to :user
end
