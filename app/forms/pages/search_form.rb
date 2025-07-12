# typed: strict
# frozen_string_literal: true

module Pages
  class SearchForm < ApplicationForm
    extend T::Sig

    attribute :q, :string

    validates :q, length: { 
      minimum: 2, 
      maximum: 100, 
      allow_blank: true,
      too_short: "は%{count}文字以上で入力してください",
      too_long: "は%{count}文字以内で入力してください"
    }
    validates :q, format: { with: /\A[^<>]*\z/, message: "不正な文字が含まれています" }

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