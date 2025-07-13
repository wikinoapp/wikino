# typed: strict
# frozen_string_literal: true

module Pages
  class SearchForm < ApplicationForm
    attribute :q, :string

    validates :q, length: {
      minimum: 2,
      maximum: 100,
      allow_blank: true,
      too_short: :too_short,
      too_long: :too_long
    }
    validates :q, format: {with: /\A[^<>]*\z/, message: :invalid}

    sig { returns(T::Boolean) }
    def q_present?
      !!q&.present?
    end

    sig { returns(T::Boolean) }
    def searchable?
      valid_result = valid?
      return false unless valid_result == true

      q_present?
    end
  end
end
