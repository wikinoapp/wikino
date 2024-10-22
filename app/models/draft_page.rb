# typed: strict
# frozen_string_literal: true

class DraftPage < ApplicationRecord
  include ModelConcerns::Pageable

  belongs_to :space
  belongs_to :topic
  belongs_to :page
  belongs_to :editor, class_name: "User"
end
