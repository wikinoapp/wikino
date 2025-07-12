# typed: strict
# frozen_string_literal: true

module Pages
  class SearchForm < ApplicationForm
    extend T::Sig

    attribute :keyword, :string

    validates :keyword, length: { 
      minimum: 2, 
      maximum: 100, 
      allow_blank: true,
      too_short: "は%{count}文字以上で入力してください",
      too_long: "は%{count}文字以内で入力してください"
    }
    validates :keyword, format: { with: /\A[^<>]*\z/, message: "不正な文字が含まれています" }

    sig { returns(T::Boolean) }
    def keyword_present?
      !!keyword&.present?
    end

    sig { returns(T::Boolean) }
    def searchable?
      valid_result = valid?
      return false unless valid_result == true
      
      keyword_present?
    end
  end
end