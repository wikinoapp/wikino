# typed: strict
# frozen_string_literal: true

class NewTopicForm < ApplicationForm
  attribute :name, :string
  attribute :description, :string, default: ""
  attribute :visibility, :string

  validates :name, presence: true
  validates :visibility, presence: true
end
