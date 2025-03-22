# typed: strict
# frozen_string_literal: true

module FormConcerns
  module UserAtnameValidatable
    extend ActiveSupport::Concern

    included do
      validates :atname,
        format: {with: User::ATNAME_FORMAT},
        length: {maximum: User::ATNAME_MAX_LENGTH},
        presence: true,
        unreserved_atname: true
    end
  end
end
