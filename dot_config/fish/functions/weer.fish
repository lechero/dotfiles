function weer --description 'Dutch weather and rain radar maps, in kitty'
    kitten icat \
        'https://cdn.knmi.nl/knmi/map/general/weather-map.gif' \
        'https://image.buienradar.nl/2.0/image/single/RadarMapRainNL?height=512&width=500&renderBackground=True&renderText=True'
end
