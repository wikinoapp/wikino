# frozen_string_literal: true

# == Schema Information
#
# Table name: email_confirmations
#
#  id         :uuid             not null, primary key
#  email      :citext           not null
#  event      :integer          not null
#  expires_at :datetime         not null
#  token      :string           not null
#  created_at :datetime         not null
#  updated_at :datetime         not null
#
# Indexes
#
#  index_email_confirmations_on_token  (token) UNIQUE
#
class EmailConfirmation < ApplicationRecord
  enum event: %i(sign_in sign_up)

  attribute :state, :string

  validates :email, presence: true, email: true
  validates :event, presence: true
  validates :token, presence: true
  validates :expires_at, presence: true
  validates :state, presence: true

  def save_and_send_email
    self.token = SecureRandom.uuid
    self.expires_at = Time.zone.now + 2.hours

    if sign_up? && User.only_kept.where(email: email).exists?
      self.event = :sign_in
    end

    if invalid?
      return self
    end

    save(validate: false)

    EmailConfirmationMailer.confirmation(id, state).deliver_now

    self
  end

  def url
    @url ||= "#{ENV.fetch('NONOTO_URL')}/callback?token=#{token}&state=#{state}"
  end

  def subject
    @subject ||= I18n.t("email_confirmation_mailer.#{event}_event.subject")
  end
end
