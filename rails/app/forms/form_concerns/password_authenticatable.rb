# typed: strict
# frozen_string_literal: true

module FormConcerns
  module PasswordAuthenticatable
    extend ActiveSupport::Concern

    included do
      validate do |record|
        next if user_record.nil?

        unless user_record.user_password_record&.authenticate(password)
          errors.add(:base, :unauthenticated)
        end
      end
    end
  end
end
