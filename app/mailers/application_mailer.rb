# typed: strict
# frozen_string_literal: true

class ApplicationMailer < ActionMailer::Base
  default from: "Nonoto <no-reply@nonoto.app>"
  layout "mailer"
end
