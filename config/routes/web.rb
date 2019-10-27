# frozen_string_literal: true

root "home#show", constraints: MemberConstraint.new
root "welcome#show", constraints: GuestConstraint.new, as: nil # Set :as option to avoid two routes with the same name
