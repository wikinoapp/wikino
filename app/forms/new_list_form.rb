# typed: strict
# frozen_string_literal: true

class NewListForm < ApplicationForm
  sig { returns(User) }
  attr_accessor :viewer

  attribute :name, :string
  attribute :description, :string, default: ""
  attribute :visibility, :string

  validates :name, presence: true
  validates :visibility, presence: true
end
