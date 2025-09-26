# typed: strict
# frozen_string_literal: true

module EditSuggestions
  class CreateForm < ApplicationForm
    attribute :title, :string
    attribute :description, :string, default: ""
    attribute :page_title, :string
    attribute :page_body, :string

    validates :title, presence: true, length: {maximum: 255}
    validates :description, length: {maximum: 10_000}
    validates :page_title, presence: true
    validates :page_body, presence: true
  end
end
