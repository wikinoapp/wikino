# frozen_string_literal: true

# class SignInTokenStrategy < Warden::Strategies::Base
#   def valid?
#     params[:sign_in_token].present?
#   end

#   def authenticate!
#     user = User.find_by(sign_in_token: params[:sign_in_token])

#     if user
#       success!(user)
#     else
#       fail!('strategies.sign_in_token.failed')
#     end
#   end
# end

# Warden::Strategies.add(:sign_in_token, SignInTokenStrategy)
