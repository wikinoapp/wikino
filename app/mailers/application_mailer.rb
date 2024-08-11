# typed: strict
# frozen_string_literal: true

class ApplicationMailer < ActionMailer::Base
  extend T::Sig

  default from: "Nonoto <no-reply@#{Nonoto.config.email_domain}>"
end
