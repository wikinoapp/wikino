# typed: strict
# frozen_string_literal: true

class ApplicationMailer < ActionMailer::Base
  extend T::Sig

  default from: "Wikino <no-reply@#{Wikino.config.email_domain}>"
end
