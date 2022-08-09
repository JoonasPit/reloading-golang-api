function firstFunction(){
    var foo = document.getElementById("stockValue").value
    var fullstring = `fetch?stockToFetch=${foo}`
    
    window.location.replace(fullstring)
}

function searchFunction(){
    var foo = document.getElementById("stockSymbol").value
    var fullstring = `by-stocksymbol?stockSymbol=${foo}`
    
    window.location.replace(fullstring)
}
function fooFunction(){
   var foo = document.getElementById("title").value
   console.log(foo)
}