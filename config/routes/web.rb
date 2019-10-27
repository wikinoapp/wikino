# frozen_string_literal: true

scope :auth do
  scope :google_oauth2 do
    resource :callback, only: %i(show), controller: :oauth_callbacks
  end
end

root "home#show", constraints: MemberConstraint.new
root "welcome#show", constraints: GuestConstraint.new, as: nil # Set :as option to avoid two routes with the same name
