# typed: strict
# frozen_string_literal: true

class PageEditorship < ApplicationRecord
  belongs_to :space
  belongs_to :page
  belongs_to :editor, class_name: "User"
end
