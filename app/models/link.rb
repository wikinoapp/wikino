# typed: strict
# frozen_string_literal: true

class Link < ApplicationRecord
  belongs_to :note
  belongs_to :target_note, class_name: "Note"
end
