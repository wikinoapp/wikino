# typed: strict
# frozen_string_literal: true

class EditTopicForm < ApplicationForm
  include FormConcerns::TopicNameValidatable

  sig { returns(T.nilable(Space)) }
  attr_accessor :space

  attribute :name, :string
  attribute :description, :string
  attribute :visibility, :string

  validates :visibility, presence: true
end
